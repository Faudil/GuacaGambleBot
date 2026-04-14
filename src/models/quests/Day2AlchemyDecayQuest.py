from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward

class Day2AlchemyDecayQuest(Quest):
    @property
    def id(self) -> str:
        return "day2_alchemy"

    @property
    def title_key(self) -> str:
        return "quests.day2_alchemy.title"

    @property
    def description_key(self) -> str:
        return "quests.day2_alchemy.description"

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day2_alchemy.step0_dialogue",
                rewards=QuestReward(money=1000)
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.day2_alchemy.step1_activity",
                rewards=QuestReward(xp=150),
                target_stat="crops_planted",
                target_count=10
            ),
            QuestStep(
                step_type=QuestStepType.CHOICE,
                text_key="quests.day2_alchemy.step2_choice",
                choices=[
                    {
                        "id": "choice_a",
                        "text_ref": "quests.day2_alchemy.choice_a",
                        "response_ref": "quests.day2_alchemy.choice_a_response"
                    },
                    {
                        "id": "choice_b",
                        "text_ref": "quests.day2_alchemy.choice_b",
                        "response_ref": "quests.day2_alchemy.choice_b_response"
                    }
                ],
                rewards=QuestReward(money=200)
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx != 1:
            return False
            
        step = self.steps[step_idx]
        if activity != step.extra.get('target_stat'):
            return False
            
        from src.database.quest import update_quest_progress
        new_val = current_progress.get('progress_value', 0) + amount
        update_quest_progress(user_id, self.id, progress_value=new_val)
        
        return new_val >= step.extra.get('target_count', 10)

    def on_choice(self, user_id: int, choice_id: str, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx == 2:
            return True
        return False

    def get_unlocks(self) -> List[str]:
        return ["day3_base"]
