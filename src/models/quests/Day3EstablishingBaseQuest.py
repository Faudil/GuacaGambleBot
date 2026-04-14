from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward

class Day3EstablishingBaseQuest(Quest):
    @property
    def id(self) -> str:
        return "day3_base"

    @property
    def title_key(self) -> str:
        return "quests.day3_base.title"

    @property
    def description_key(self) -> str:
        return "quests.day3_base.description"

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.REQUIREMENT,
                text_key="quests.day3_base.step0_req",
                rewards=QuestReward(xp=100),
                requirements={"money": 500, "items": {"minerai de fer": 2}},
                consume_reqs=True
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.day3_base.step1_activity",
                rewards=QuestReward(xp=200),
                target_stat="house_bought",
                target_count=1
            ),
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day3_base.step2_dialogue",
                rewards=QuestReward(money=100)
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
        
        return new_val >= step.extra.get('target_count', 1)

    def get_unlocks(self) -> List[str]:
        return ["day4_will"]
