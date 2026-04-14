from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward, QuestType, QuestRegistry

class DiggingUpThePastQuest(Quest):
    id = "digging_up_past"
    type = QuestType.MAIN

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.digging_up_past.step0_thorek",
                rewards=QuestReward(money=500) # Money to buy a pickaxe
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.digging_up_past.step1_mine",
                rewards=QuestReward(items=["stone", "stone"], xp=150),
                target_stat="items_mined",
                target_count=1
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx != 1:
            return False
            
        step = self.steps[step_idx]
        target_stat = step.extra.get('target_stat')
        if activity != target_stat:
            return False
            
        new_val = current_progress.get('progress_value', 0) + amount
        from src.database.quest import update_quest_progress
        update_quest_progress(user_id, self.id, progress_value=new_val)
        
        return new_val >= step.extra.get('target_count', 1)

    def get_unlocks(self) -> List[str]:
        return ["scrap_farm"]

# Register the quest
QuestRegistry.register(DiggingUpThePastQuest)
